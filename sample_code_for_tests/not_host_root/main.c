#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

int main(void) {
    const char *content = "sandbox-hostname\n";
    int fd = open("/proc/sys/kernel/hostname", O_WRONLY);
    if (fd < 0) {
        printf("not host root: ok\n");
        return 0;
    }

    ssize_t written = write(fd, content, strlen(content));
    close(fd);
    if (written >= 0) {
        fprintf(stderr, "is host root: bad\n");
        return 0;
    }

    if (errno == EPERM || errno == EACCES || errno == EROFS) {
        printf("not host root: ok\n");
        return 0;
    }

    perror("write");
    return 1;
}
