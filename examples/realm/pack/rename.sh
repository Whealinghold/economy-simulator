#!/bin/sh
# Regenerates pack/cold and pack/modules from the layers of testdata/chains by the
# whole-token rename of names.txt (plan 2-24 D682). Full-line comments are dropped; a
# bare key at the start of a line (haul = "haul") is a schema key and is not renamed.
# Run from examples/realm/pack. The order of the module layers is the allLayers order
# of all_pack_test.go in the root module, without the warm start (D684).
set -eu
src=../../../testdata/chains
rename() {
	perl -e '
		open my $n, "<", "names.txt" or die; my %m;
		while (<$n>) { next if /^#/; my ($o, $w) = split; $m{$o} = $w if defined $w }
		my $re = join "|", map { quotemeta } sort { length($b) <=> length($a) } keys %m;
		print "# Generated from testdata/chains/$ARGV[0] by rename.sh; do not edit.\n";
		open my $f, "<", "$ARGV[1]" or die;
		while (<$f>) { next if /^\s*#/; my ($key, $rest) = /^(\s*\w+\s*=)(.*)$/s ? ($1, $2) : ("", $_); $rest =~ s/(?<![A-Za-z0-9_])($re)(?![A-Za-z0-9_])/$m{$1}/g; print $key, $rest }
	' "$1" "$src/$1" >"$2"
}
mkdir -p cold modules
for l in base lines five; do rename "$l.toml" "cold/$l.toml"; done
n=0
for l in overlay/trade-five.toml all/demand.toml overlay/faucet.toml overlay/population.toml \
	overlay/transport-five.toml overlay/resources-five.toml overlay/information-five.toml \
	overlay/external.toml all/subsistence.toml overlay/migration.toml all/merchants.toml \
	overlay/estates.toml; do
	n=$((n + 1))
	b=$(basename "$l" .toml)
	rename "$l" "$(printf 'modules/%02d-%s.toml' "$n" "$b")"
done
