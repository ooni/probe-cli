#include <stdlib.h>

#include <fstream>
#include <iostream>
#include <iterator>
#include <string>

#define OONIFFI_EMULATE_MK_API
#include "ooniffi.h"

int main(int argc, char **argv) {
  if (argc != 2) {
    std::clog << "usage: ffirun /path/to/json/settings" << std::endl;
    exit(1);
  }
  std::ifstream filep(argv[1]);
  if (!filep.good()) {
    std::clog << "fatal: cannot open settings file" << std::endl;
    exit(1);
  }
  std::string settings((std::istreambuf_iterator<char>(filep)),
      std::istreambuf_iterator<char>());
  auto taskp = mk_task_start(settings.c_str());
  if (taskp == nullptr) {
    std::clog << "fatal: cannot start task" << std::endl;
    exit(1);
  }
  while (!mk_task_is_done(taskp)) {
    auto evp = mk_task_wait_for_next_event(taskp);
    if (evp == nullptr) {
      std::clog << "warning: cannot wait for next event" << std::endl;
      break;
    }
    auto evstr = mk_event_serialization(evp);
    if (evstr != nullptr) {
      std::cout << evstr << std::endl;
    } else {
      std::clog << "warning: cannot get event serialization" << std::endl;
    }
    mk_event_destroy(evp);
  }
  mk_task_destroy(taskp);
}
