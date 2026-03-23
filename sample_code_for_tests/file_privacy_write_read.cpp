#include <stdio.h>
#include <sys/stat.h>

int main() {
    mkdir("/root", 0700);
    FILE *f = fopen("/root/test.txt", "w");
    if (!f) {
        perror("fopen write");
        return 1;
    }

    fprintf(f, "SecretData123");
    fclose(f);

    char buf[64] = {0};
    f = fopen("/root/test.txt", "r");
    if (!f) {
        perror("fopen read");
        return 1;
    }

    fgets(buf, sizeof(buf), f);
    fclose(f);
    printf("%s", buf);
    return 0;
}
