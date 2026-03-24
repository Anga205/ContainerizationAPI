buf = b'A' * (1024 * 1024)

with open('/disk_spam.bin', 'wb', buffering=0) as f:
    while True:
        f.write(buf)
        f.flush()
