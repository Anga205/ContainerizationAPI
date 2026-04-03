#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <unistd.h>

int main(void) {
    int fd = open("/etc/passwd", O_WRONLY);
    if (fd >= 0) {
        fprintf(stderr, "write succeeded\n");
        close(fd);
        return 0;
    }

    if (errno == EACCES || errno == EPERM || errno == EROFS || errno == ENOENT) {
        printf("write correctly denied\n");
        return 0;
    }

    perror("open");
    return 1;
}
