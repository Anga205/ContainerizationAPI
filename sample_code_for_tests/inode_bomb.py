import sys

for i in range(10000):
    with open(f'file_{i}.txt', 'w', encoding='utf-8'):
        pass

print('inode bomb completed', file=sys.stderr)
