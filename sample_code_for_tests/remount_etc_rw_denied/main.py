import ctypes
import errno
import os
import sys

MS_BIND = 4096
MS_REMOUNT = 32

libc = ctypes.CDLL(None, use_errno=True)
result = libc.mount(ctypes.c_char_p(0), b"/etc", ctypes.c_char_p(0), MS_BIND | MS_REMOUNT, ctypes.c_char_p(0))
if result == 0:
    print("remount succeeded", file=sys.stderr)
    raise SystemExit(0)

err = ctypes.get_errno()
if err in (errno.EPERM, errno.EACCES, errno.EROFS):
    print("remount correctly denied")
    raise SystemExit(0)

print(os.strerror(err), file=sys.stderr)
raise SystemExit(1)
