#include <stdio.h>
#include <unistd.h>

int main() {
    char line[4096];
    int n = snprintf(line, sizeof(line), "IO_FLOOD:");
    for (int i = n; i < (int)sizeof(line) - 2; i++) {
        line[i] = 'A';
    }
    line[sizeof(line) - 2] = '\n';
    line[sizeof(line) - 1] = '\0';

    while (1) {
        fputs(line, stderr);
        fflush(stderr);
        usleep(1000);
    }
}
