import ctypes
import os
import time

PR_SET_NAME = 15
libc = ctypes.CDLL(None)

child = os.fork()
if child == 0:
    grandchild = os.fork()
    if grandchild == 0:
        libc.prctl(PR_SET_NAME, b'orphanpygc', 0, 0, 0)
        time.sleep(60)
        raise SystemExit(0)
    raise SystemExit(0)

raise SystemExit(0)
