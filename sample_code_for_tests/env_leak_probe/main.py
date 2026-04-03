import os

allowed = {"PATH", "HOME", "LANG", "LD_LIBRARY_PATH", "JAVA_HOME"}

for name in sorted(os.environ):
    if name not in allowed:
        print(f"ENV_LEAK:{name}")
