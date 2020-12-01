#ifndef _WIN32
#include <syslog.h>
#endif

void ooniprobe_openlog(void) {
#ifndef _WIN32
    (void)openlog("ooniprobe", LOG_PID, LOG_USER);
#endif
}

void ooniprobe_syslog(int level, const char *message) {
#ifndef _WIN32
    (void)syslog(level, "%s", message);
#endif
}
