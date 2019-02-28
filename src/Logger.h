#ifndef _LOGGER_H_
#define _LOGGER_H_

#include "common.h"
#include <log4cpp/CategoryStream.hh>

namespace flocons {

class Logger {
  public:
    class Category {
        friend class Logger;

      private:
        std::string _name;
        Category(const std::string& name);

      public:
        const std::string& name() { return this->_name; }
        std::ostream error;
        std::ostream warn;
        std::ostream info;
        std::ostream debug;
    };

  public:
    static std::shared_ptr<Category> category(const std::string& name);
    static std::ostream error;
    static std::ostream warn;
    static std::ostream info;
    static std::ostream debug;
};

} // namespace flocons

#endif // !_LOGGER_H_