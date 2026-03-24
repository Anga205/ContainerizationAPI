#include <stdio.h>
#include <string.h>

int main() {
    static char buf[1024 * 1024];
    memset(buf, 'A', sizeof(buf));

    FILE *f = fopen("/disk_spam.bin", "w");
    if (!f) {
        perror("fopen");
        return 1;
    }

    while (1) {
        if (fwrite(buf, 1, sizeof(buf), f) != sizeof(buf)) {
            perror("fwrite");
            fflush(f);
            return 1;
        }
        fflush(f);
    }
}
