#ifndef _FLOCONS_FILE_DATA_CONTAINER_REGULAR_FILE_ACCESSOR_H_
#define _FLOCONS_FILE_DATA_CONTAINER_REGULAR_FILE_ACCESSOR_H_

#include "../../common.h"
#include "RegularFileAccessor.h"

namespace flocons {

class DataContainerRegularFileAccessor : public RegularFileAccessor {
  private:
    char* _data;

  public:
    DataContainerRegularFileAccessor(const char* data, size_t size) : RegularFileAccessor(), _data(memdup(data, size)) {}
    virtual ~DataContainerRegularFileAccessor() { free(this->_data); }
    virtual char* data() { return this->_data; }
};

} // namespace flocons

#endif // !_FLOCONS_FILE_DATA_CONTAINER_REGULAR_FILE_ACCESSOR_H_