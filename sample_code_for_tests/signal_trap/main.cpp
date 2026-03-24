#include <signal.h>
#include <unistd.h>

int main() {
    signal(SIGTERM, SIG_IGN);
    signal(SIGINT, SIG_IGN);

    while (1) {
        pause();
    }
}
