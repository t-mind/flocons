#ifndef _FLOCONS_HTTP_REQUEST_POOL_H_
#define _FLOCONS_HTTP_REQUEST_POOL_H_

#include "../common.h"
#include "HTTPRequest.h"

namespace flocons {

class HTTPRequestPool {
  private:
    std::map<std::string, std::vector<std::shared_ptr<HTTPRequest>>> _pools;

  public:
    std::shared_ptr<HTTPRequest> getAndLockRequest(const URL& base_url);
};

} // namespace flocons

#endif // !_FLOCONS_HTTP_REQUEST_POOL_H_