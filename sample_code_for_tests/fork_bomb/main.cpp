#include <stdio.h>
#include <sys/types.h>
#include <unistd.h>

int main() {
    while (1) {
        pid_t p = fork();
        if (p < 0) {
            perror("fork");
            return 0;
        }

        if (p == 0) {
            for (;;) {
                pause();
            }
        }
    }
}
