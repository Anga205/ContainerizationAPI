#define _GNU_SOURCE
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <sys/types.h>
#include <unistd.h>

int main() {
    const size_t chunk = 1024 * 1024;

    for (int i = 0; i < 16; i++) {
        pid_t p = fork();
        if (p == 0) {
            while (1) {
                void *ptr = mmap(NULL, chunk, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
                memset(ptr, 0xAB, chunk);
            }
        }
    }

    while (1) {
        pause();
    }
}
