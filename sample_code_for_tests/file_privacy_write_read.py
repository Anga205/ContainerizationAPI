import os

os.makedirs('/root', exist_ok=True)
with open('/root/test.txt', 'w', encoding='utf-8') as f:
    f.write('SecretData123')

with open('/root/test.txt', 'r', encoding='utf-8') as f:
    print(f.read(), end='')
