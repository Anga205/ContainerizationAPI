import sys

try:
    with open("/proc/sys/kernel/hostname", "w", encoding="utf-8") as handle:
        handle.write("sandbox-hostname\n")
except OSError:
    print("not host root: ok")
    raise SystemExit(0)

print("is host root: bad", file=sys.stderr)
raise SystemExit(0)
