"""The byte-level examples of spec-checkpoint.md 17, and the arithmetic they rest on.

A model of the encoding rules of spec-checkpoint.md 5, not code of the library. Standard
library only. The Go values that the spec quotes were printed by Go 1.25 in Docker and are
hard-coded here as the expected results.

Run in Docker: docker run --rm -v "$PWD":/w -w /w python:3.12-slim python worked.py
"""

import hashlib

M64 = (1 << 64) - 1
GAMMA = 0x9E3779B97F4A7C15
KEY = 0x243F6A8885A308D3


# --- the primitives of 5.2 -------------------------------------------------------------

def u(v):
    assert 0 <= v <= M64
    out = bytearray()
    while v >= 0x80:
        out.append((v & 0x7F) | 0x80)
        v >>= 7
    out.append(v)
    return bytes(out)


def i(v):
    assert -(1 << 63) <= v < (1 << 63)
    return u(((v << 1) ^ (v >> 63)) & M64)


def b(v):
    return bytes([1 if v else 0])


class Corrupt(Exception):
    pass


def read_u(buf, pos):
    """Minimal-form LEB128; the check Go's binary.Uvarint does not make."""
    v, shift, start = 0, 0, pos
    while True:
        if pos >= len(buf):
            raise Corrupt("truncated varint")
        c = buf[pos]
        pos += 1
        if shift == 63 and c > 1:
            raise Corrupt("varint overflows 64 bits")
        v |= (c & 0x7F) << shift
        if c < 0x80:
            break
        shift += 7
        if shift > 63:
            raise Corrupt("varint longer than 10 bytes")
    if pos - start > 1 and buf[pos - 1] == 0:
        raise Corrupt("non-minimal varint")
    return v, pos


def read_i(buf, pos):
    z, pos = read_u(buf, pos)
    return (z >> 1) ^ -(z & 1), pos


def read_b(buf, pos):
    if pos >= len(buf) or buf[pos] > 1:
        raise Corrupt("bad boolean")
    return buf[pos] == 1, pos + 1


# --- the world's source (spec-world.md 7.1), for the example state --------------------------

def finalise(z):
    z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
    z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
    return z ^ (z >> 31)


class PCG:
    MUL = (2549297995355413924 << 64) | 4865540595714422341
    INC = (6364136223846793005 << 64) | 1442695040888963407

    def __init__(self, hi, lo):
        self.state = (hi << 64) | lo

    def uint64(self):
        self.state = (self.state * self.MUL + self.INC) & ((1 << 128) - 1)
        hi, lo = self.state >> 64, self.state & M64
        hi ^= hi >> 32
        hi = (hi * 0xDA942042E4DD58B5) & M64
        hi ^= hi >> 48
        return (hi * (lo | 1)) & M64

    def marshal(self):
        return b"pcg:" + (self.state >> 64).to_bytes(8, "big") + (self.state & M64).to_bytes(8, "big")


def seed_state(seed):
    s = (seed + GAMMA) & M64
    hi = finalise(s)
    s = (s + GAMMA) & M64
    return hi, finalise(s)


def reseed(hi, lo, salt):
    x = finalise(salt ^ KEY)
    return finalise(hi ^ x), finalise(lo ^ finalise((x + GAMMA) & M64))


def main():
    # 17.1 primitives
    cases_u = [0, 1, 127, 128, 300, 16383, 16384, 2**63, M64]
    print("unsigned:")
    for v in cases_u:
        print("  U(%d) = %s" % (v, u(v).hex(" ")))
    print("signed (zigzag):")
    for v in [0, -1, 1, -64, 64, 10000, -(1 << 63), (1 << 63) - 1]:
        print("  I(%d) = %s" % (v, i(v).hex(" ")))
    for v in cases_u:
        assert read_u(u(v), 0) == (v, len(u(v)))
    for v in [0, -1, 1, -64, 64, 10000, -(1 << 63), (1 << 63) - 1]:
        assert read_i(i(v), 0) == (v, len(i(v)))
    assert i(-(1 << 63)) == b"\xff" * 9 + b"\x01"

    print("rejected inputs:")
    for name, raw in [
        ("80 00 (0 in two bytes)", bytes.fromhex("8000")),
        ("81 80 00 (1 in three bytes)", bytes.fromhex("818000")),
        ("ff*9 02 (2^64 + ...)", bytes.fromhex("ff" * 9 + "02")),
        ("80 (truncated)", bytes.fromhex("80")),
    ]:
        try:
            read_u(raw, 0)
            raise SystemExit("accepted " + name)
        except Corrupt as e:
            print("  %-30s -> %s" % (name, e))

    # 17.2 the world section of the construction of spec-world.md 19.1 (seed 0, 14 words)
    hi, lo = seed_state(0)
    p = PCG(hi, lo)
    for _ in range(14):
        p.uint64()
    state = p.marshal()
    assert state.hex() == "7063673a8e3e6b7eff8db56cb58e6a2f88e0bbfa", state.hex()
    assert "%016x" % p.uint64() == "212416c8026d6a45"
    # a fresh PCG of seed 0 for the reseed vectors is the state after the 14 words
    hi14, lo14 = p.state >> 64, p.state & M64
    # p advanced one more word above; rebuild the state after 14 words for the section
    q = PCG(hi, lo)
    for _ in range(14):
        q.uint64()
    body = q.marshal() + u(11) + b(True)
    section = u(1) + u(len(body)) + body
    print("world section (tag 1): %s" % section.hex(" "))

    # 17.3 a toy frame: header, the world section, trailer
    magic = b"ESCK"
    cfg_hash = bytes(range(32))
    header = magic + u(1) + u(1) + cfg_hash + u(0) + u(0)
    frame = header + section
    digest = hashlib.sha256(frame).digest()
    print("toy frame length %d, digest %s" % (len(frame) + 32, digest.hex()))
    print("header: %s" % header.hex(" "))

    # 17.4 the account of an agent
    acct = u(0) + b(False) + u(1) + u(4) + i(10000) + u(1) + u(3) + u(5) + i(0) + u(0)
    print("agent account: %s" % acct.hex(" "))
    closed = u(0) + b(True)
    print("closed account: %s" % closed.hex(" "))

    # 17.5 reseed vectors from the state after 14 words
    hi14, lo14 = q.state >> 64, q.state & M64
    print("state words hi=%016x lo=%016x" % (hi14, lo14))
    expected = {
        0: ("0eec475500551da4", "64b50bad715e9bc6", "51798660ccefa5fa"),
        1: ("8d300de17fda20a5", "52b5dbda40cf330c", "081f02238dd84e8c"),
        2: ("c44be769bde5c5d0", "7603e7237b5b0efa", "e7b2f5564db971e6"),
        42: ("26e84dab9a52f23d", "ca6855e42adbe836", "80c06a62e31cb550"),
    }
    for salt, want in expected.items():
        h, l = reseed(hi14, lo14, salt)
        first = PCG(h, l).uint64()
        got = ("%016x" % h, "%016x" % l, "%016x" % first)
        assert got == want, (salt, got, want)
        print("  salt %d: hi=%s lo=%s first word=%s" % (salt, *got))
    print("reseed matches Go 1.25 for salts", sorted(expected))


if __name__ == "__main__":
    main()
