"""The byte-level examples of spec-checkpoint.md 17, and the arithmetic they rest on.

A model of the encoding rules of spec-checkpoint.md 5, not code of the library. Standard
library only. The Go values that the spec quotes were printed by Go 1.25 in Docker and are
hard-coded here as the expected results.

Run in Docker, never on the host, from the models folder so that wld/seed.py is importable
(the finaliser, the key and the reseed rule are that file's, one source):
  docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/..":/m -w /m/chk python:3.12-slim python worked.py

The world section of 17.2 holds the next agent ID and the construction flag only (ADR 0008:
a checkpoint saves the seed, in the header, and no stream state). The reseed of 6.8 and 17.5
is the rule of ADR 0008 and is computed by wld/seed.py.
"""

import hashlib
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "wld"))

from seed import SITES, M64, reseed, stream_state, words  # noqa: E402

GAMMA = 0x9E3779B97F4A7C15


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

    # 17.2 the world section of the construction of spec-world.md 19.1: seven household agents (IDs 4 to 10),
    # so the next agent ID is 11, and the world has not stepped, so the construction flag is set
    body = u(11) + b(True)
    section = u(1) + u(len(body)) + body
    if section.hex() != "01020b01":
        raise AssertionError(section.hex())
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

    # 17.5 the reseed of ADR 0008: seed' = finalise(seed xor finalise(salt)), checked against the values
    # printed by a Go program (stdlib math/rand/v2 PCG, Go 1.25 and 1.27: not archived)
    expected = {
        (0, 0): ("0000000000000000", "264d986499a1a1fa"),
        (0, 1): ("7ab40e090f363a7d", "9fea3673fd1e605f"),
        (0, 2): ("52ead7e36ea7fea8", "73a2087f8eae98d5"),
        (0, 42): ("97ea87f7e45c00a5", "1b4989fe2e01283b"),
        (1, 0): ("5692161d100b05e5", "3a90ed51beb4e41d"),
        (1, 1): ("83ec686c1600460a", "a5d0a8b51be8378f"),
        (1, 2): ("3ec2d42f3a45cc6e", "47f38e4d9854e8fd"),
        (1, 42): ("901883d2427428f4", "00c073c4d5d6c31a"),
        (42, 0): ("a759ea27d4727622", "e3082397f3bc8ba6"),
        (42, 1): ("c2a6eebdf3976ad0", "cd5e38f87f0ec634"),
        (42, 2): ("7bfa87c92aa0cff0", "5b73a5b88ec53d5a"),
        (42, 42): ("36cdb0593534a37b", "472c27560611be0f"),
        (18446744073709551615, 0): ("b4d055fcf2cbbd7b", "9cbc3a9d395aed3d"),
        (18446744073709551615, 1): ("f3759af4852e5904", "e6b61a2920e28083"),
        (18446744073709551615, 2): ("ab47721b00f39ddb", "649e9d647ec06743"),
        (18446744073709551615, 42): ("eab06b8e3b2074d7", "71829937eb038fa1"),
    }
    print("reseed, seed' = finalise(seed xor finalise(salt)); w0 is the first word of the stream (siteWork, 0, 0) of seed'")
    for (seed, salt), (want_seed, want_w0) in expected.items():
        s2 = reseed(seed, salt)
        hi, lo = stream_state(s2, SITES["siteWork"], 0, 0)
        got = ("%016x" % s2, "%016x" % words(hi, lo, 1)[0])
        assert got == (want_seed, want_w0), (seed, salt, got)
        print("  seed %d salt %d: seed'=%s w0=%s" % (seed, salt, got[0], got[1]))
    assert reseed(0, 0) == 0  # finalise(0) = 0: the one known case where a reseed is the identity
    print("reseed matches Go for 16 (seed, salt) pairs")


if __name__ == "__main__":
    main()
