#include "HTTPRequestPool.h"

namespace flocons {

std::shared_ptr<HTTPRequest> HTTPRequestPool::getAndLockRequest(const URL& base_url) {
    std::vector<std::shared_ptr<HTTPRequest>>& pool = this->_pools[base_url.string()];
    for (auto it = pool.begin(); it != pool.end(); ++it) {
        if ((*it)->try_lock())
            return *it;
    }
    auto request = std::make_shared<HTTPRequest>(base_url);
    request->try_lock();
    pool.push_back(request);
    return request;
}

} // namespace flocons