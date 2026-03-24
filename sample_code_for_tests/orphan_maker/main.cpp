#include <stdio.h>
#include <sys/prctl.h>
#include <sys/types.h>
#include <unistd.h>

int main() {
    pid_t child = fork();
    if (child < 0) {
        perror("fork child");
        return 1;
    }

    if (child == 0) {
        pid_t grandchild = fork();
        if (grandchild < 0) {
            perror("fork grandchild");
            return 1;
        }

        if (grandchild == 0) {
            prctl(PR_SET_NAME, "orphanmakergc", 0, 0, 0);
            for (int i = 0; i < 60; i++) {
                sleep(1);
            }
            return 0;
        }

        return 0;
    }

    return 0;
}
