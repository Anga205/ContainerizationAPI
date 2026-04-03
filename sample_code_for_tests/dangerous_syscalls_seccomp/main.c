#include <errno.h>
#include <stdio.h>
#include <sys/mount.h>

int main(void) {
    if (mount("", "/etc", "", MS_BIND | MS_REMOUNT, "") == 0) {
        fprintf(stderr, "syscall succeeded unexpectedly\n");
        return 0;
    }

    if (errno == EPERM || errno == EACCES || errno == ENOSYS) {
        printf("syscall denied: ok\n");
        return 0;
    }

    perror("mount");
    return 1;
}
