#include <stdio.h>

int main() {
    for (int i = 0; i < 10000; i++) {
        char name[64];
        snprintf(name, sizeof(name), "file_%d.txt", i);

        FILE *f = fopen(name, "w");
        if (!f) {
            perror("fopen");
            return 1;
        }
        fclose(f);
    }

    fprintf(stderr, "inode bomb completed\n");
    return 0;
}
