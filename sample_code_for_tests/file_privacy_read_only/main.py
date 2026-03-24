import sys

try:
    with open('/root/test.txt', 'r', encoding='utf-8') as f:
        print(f.read(), end='')
except FileNotFoundError as exc:
    print(exc, file=sys.stderr)
    raise SystemExit(1)
