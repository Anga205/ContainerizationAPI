#include <stdio.h>

int main() {
    FILE *f = fopen("/root/test.txt", "r");
    if (!f) {
        perror("fopen");
        return 1;
    }

    char buf[64] = {0};
    fgets(buf, sizeof(buf), f);
    fclose(f);
    printf("%s", buf);
    return 0;
}
