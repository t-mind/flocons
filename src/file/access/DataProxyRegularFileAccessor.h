#ifndef _FLOCONS_FILE_DATA_PROXY_REGULAR_FILE_ACCESSOR_H_
#define _FLOCONS_FILE_DATA_PROXY_REGULAR_FILE_ACCESSOR_H_

#include "../../common.h"
#include "RegularFileAccessor.h"

namespace flocons {

class DataProxyRegularFileAccessor : public RegularFileAccessor {
  private:
    const std::function<char*()> _data;

  public:
    DataProxyRegularFileAccessor(const std::function<char*()>& data) : _data(data) {}
    virtual char* data() {
        if (this->_data != nullptr) return this->_data();
        return NULL;
    }
};

} // namespace flocons

#endif // !_FLOCONS_FILE_DATA_PROXY_REGULAR_FILE_ACCESSOR_H_