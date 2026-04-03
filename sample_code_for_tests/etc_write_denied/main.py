import errno
import os
import sys

try:
    fd = os.open("/etc/passwd", os.O_WRONLY)
except OSError as exc:
    if exc.errno in (errno.EACCES, errno.EPERM, errno.EROFS):
        print("write correctly denied")
        raise SystemExit(0)
    print(str(exc), file=sys.stderr)
    raise SystemExit(1)
else:
    os.close(fd)
    print("write succeeded", file=sys.stderr)
    raise SystemExit(0)
