#ifndef _WIN32
#include <syslog.h>
#endif

void ooniprobe_openlog(void) {
#ifndef _WIN32
    (void)openlog("ooniprobe", LOG_PID, LOG_USER);
#endif
}

void ooniprobe_log_debug(const char *message) {
#ifndef _WIN32
    (void)syslog(LOG_DEBUG, "%s", message);
#endif
}

void ooniprobe_log_info(const char *message) {
#ifndef _WIN32
    (void)syslog(LOG_INFO, "%s", message);
#endif
}

void ooniprobe_log_warning(const char *message) {
#ifndef _WIN32
    (void)syslog(LOG_WARNING, "%s", message);
#endif
}

void ooniprobe_log_err(const char *message) {
#ifndef _WIN32
    (void)syslog(LOG_ERR, "%s", message);
#endif
}

void ooniprobe_log_crit(const char *message) {
#ifndef _WIN32
    (void)syslog(LOG_CRIT, "%s", message);
#endif
}
