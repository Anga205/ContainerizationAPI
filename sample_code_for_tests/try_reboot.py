import ctypes
import os
import sys

RB_AUTOBOOT = 0x01234567
libc = ctypes.CDLL(None, use_errno=True)
os.sync()

if libc.reboot(RB_AUTOBOOT) == 0:
    print('reboot succeeded unexpectedly', file=sys.stderr)
    raise SystemExit(0)

err = ctypes.get_errno()
print(os.strerror(err), file=sys.stderr)
raise SystemExit(1)
