import multiprocessing as mp
import time


def eater() -> None:
    chunks = []
    while True:
        chunks.append(bytearray(1024 * 1024))


if __name__ == '__main__':
    procs = []
    for _ in range(16):
        p = mp.Process(target=eater)
        p.start()
        procs.append(p)

    while True:
        time.sleep(1)
