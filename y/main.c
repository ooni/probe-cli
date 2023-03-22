#include <tor_api.h>

#include <pthread.h>
#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>

static void *threadMain(void *ptr) {
	int *fdp = (int*)ptr;
	(void)sleep(45 /* second */);
	(void)close(*fdp);
	free(fdp);
	return NULL;
}

int main() {
	for (;;) {
		tor_main_configuration_t *config = tor_main_configuration_new();
		if (config == NULL) {
			exit(1);
		}
		char *argv[] = {
			"tor",
			"Log",
			"notice stderr",
			"DataDirectory",
			"./x",
			NULL,
		};
		int argc = 5;
		if (tor_main_configuration_set_command_line(config, argc, argv) != 0) {
			exit(2);
		}
		int filedesc = tor_main_configuration_setup_control_socket(config);
		if (filedesc < 0) {
			exit(3);
		}
		int *fdp = malloc(sizeof(*fdp));
		if (fdp == NULL) {
			exit(4);
		}
		*fdp = filedesc;
		pthread_t thread;
		if (pthread_create(&thread, NULL, threadMain, /* move */ fdp) != 0) {
			exit(5);
		}
		tor_run_main(config);
		if (pthread_join(thread, NULL) != 0) {
			exit(6);
		}
		fprintf(stderr, "********** doing another round\n");
	}
}
