#ifndef _FLOCONS_FILE_REGULAR_FILE_ACCESSOR_H_
#define _FLOCONS_FILE_REGULAR_FILE_ACCESSOR_H_

#include "../../common.h"

namespace flocons {

class RegularFileAccessor {
  public:
    virtual ~RegularFileAccessor() {}
    virtual char* data() = 0;
};

} // namespace flocons

#endif // !_FLOCONS_FILE_REGULAR_FILE_ACCESSOR_H_
