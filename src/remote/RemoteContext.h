#ifndef _REMOTE_CONTEXT_H_
#define _REMOTE_CONTEXT_H_

#include "../common.h"

class RemoteContext {
  private:
    std::string _host;

  public:
    RemoteContext(const std::string& host) : _host(host) {}
    inline const std::string& host() const { return this->_host; }
};

#endif // !_REMOTE_CONTEXT_H_
