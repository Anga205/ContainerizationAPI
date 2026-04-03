#include <stdio.h>
#include <string.h>

extern char **environ;

static int is_allowed(const char *name) {
    return strcmp(name, "PATH") == 0 ||
           strcmp(name, "HOME") == 0 ||
           strcmp(name, "LANG") == 0 ||
           strcmp(name, "LD_LIBRARY_PATH") == 0 ||
           strcmp(name, "JAVA_HOME") == 0;
}

int main(void) {
    for (char **entry = environ; *entry != NULL; entry++) {
        char *equals = strchr(*entry, '=');
        if (equals == NULL) {
            continue;
        }

        size_t name_len = (size_t)(equals - *entry);
        if (name_len == 0 || name_len >= 256) {
            continue;
        }

        char name[256];
        memcpy(name, *entry, name_len);
        name[name_len] = '\0';

        if (!is_allowed(name)) {
            printf("ENV_LEAK:%s\n", name);
        }
    }

    return 0;
}
