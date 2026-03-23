#include <errno.h>
#include <linux/reboot.h>
#include <stdio.h>
#include <sys/reboot.h>
#include <unistd.h>

int main() {
    sync();
    if (reboot(RB_AUTOBOOT) == 0) {
        fprintf(stderr, "reboot succeeded unexpectedly\n");
        return 0;
    }

    perror("reboot");
    return 1;
}
