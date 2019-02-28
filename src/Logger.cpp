#include "Logger.h"
#include <log4cpp/Appender.hh>
#include <log4cpp/BasicLayout.hh>
#include <log4cpp/Category.hh>
#include <log4cpp/FileAppender.hh>
#include <log4cpp/Layout.hh>
#include <log4cpp/OstreamAppender.hh>
#include <log4cpp/Priority.hh>
#include <log4cpp/PropertyConfigurator.hh>

namespace flocons {

// We have a pool of streams by thread. log4cpp is thread safe, but our stream wrapper isn't, it is easier this way
thread_local std::map<std::string, std::shared_ptr<Logger::Category>> _categories;

static bool _initialized = false;
static std::mutex _initialize_lock;

void _initialize() {
    std::lock_guard<std::mutex> lock(_initialize_lock);
    if (_initialized) return;
    std::string initFileName = "log4cpp.properties";
    log4cpp::PropertyConfigurator::configure(initFileName);
    _initialized = true;
}

class Stream : public std::stringbuf {
  private:
    log4cpp::Priority::PriorityLevel _level;
    std::string _name;

  public:
    Stream(log4cpp::Priority::PriorityLevel level, const std::string& name = "") : _level(level), _name(name) {}
    virtual int sync() {
        if (!_initialized) _initialize();
        log4cpp::Category& cat = this->_name.empty() ? log4cpp::Category::getRoot() : log4cpp::Category::getInstance(this->_name);
        cat << this->_level << this->str();
        this->str("");
        return 0;
    }
};

Logger::Category::Category(const std::string& name)
    : _name(name), error(new Stream(log4cpp::Priority::ERROR, name)), warn(new Stream(log4cpp::Priority::WARN, name)),
      info(new Stream(log4cpp::Priority::INFO, name)), debug(new Stream(log4cpp::Priority::DEBUG, name)) {}

std::ostream Logger::error(new Stream(log4cpp::Priority::ERROR));
std::ostream Logger::warn(new Stream(log4cpp::Priority::WARN));
std::ostream Logger::info(new Stream(log4cpp::Priority::INFO));
std::ostream Logger::debug(new Stream(log4cpp::Priority::DEBUG));

std::shared_ptr<Logger::Category> Logger::category(const std::string& name) {
    auto it = _categories.find(name);
    if (it != _categories.end()) return it->second;
    std::shared_ptr<Category> category(new Category(name)); // Use allocator with new because Category constructor is private and can't be called by make_shared
    _categories[name] = category;
    return category;
}

} // namespace flocons