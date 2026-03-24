import sys
import time

line = 'IO_FLOOD:' + ('A' * 4086) + '\n'
while True:
    sys.stderr.write(line)
    sys.stderr.flush()
    time.sleep(0.001)
