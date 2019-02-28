#include "../Logger.h"
#include "../common.h"
#include "../local/HTTPFileServer.h"
#include "../local/LocalFileService.h"
#include <condition_variable>
#include <csignal>
#include <getopt.h>
#include <unistd.h>

using namespace flocons;

namespace {
volatile std::sig_atomic_t signal_status = 0;
}
std::mutex exit_mutex;
std::condition_variable exit_cond_var;

void print_usage(char* program_name) {
    std::cout << program_name << " [--verbose|-v] [--version|-V] [--help|-h] [--hostname|-H <hostname>] [--port|-p <port>] <data_folder>" << std::endl;
}

void signal_handler(int signal) {
    std::unique_lock<std::mutex> lock(exit_mutex);
    Logger::info << "Received signal " << strsignal(signal) << "(" << signal << "), exiting" << std::endl;
    signal_status = signal;
    exit_cond_var.notify_all();
}

int main(int argc, char** argv) {
    int verbose_flag = 0;

    char hostname[HOST_NAME_MAX + 1];
    gethostname(hostname, HOST_NAME_MAX + 1);

    int port = 0;
    std::string data_folder;
    std::shared_ptr<LocalFileService> file_service;
    std::shared_ptr<HTTPFileServer> file_server;

    static struct option long_options[] = {{"verbose", no_argument, 0, 'v'},        {"help", no_argument, 0, 'h'},       {"version", no_argument, 0, 'V'},
                                           {"hostname", required_argument, 0, 'H'}, {"port", required_argument, 0, 'p'}, {0, 0, 0, 0}};

    int arg_index = 1;
    int c;
    while ((c = getopt_long(argc, argv, "vVhH:p:", long_options, NULL)) != -1) {
        ++arg_index;
        Logger::debug << "Parse argument " << (char)c << " at pos " << arg_index << std::endl;
        if (optarg != NULL) {
            ++arg_index;
            Logger::debug << "Value is " << optarg << std::endl;
        }
        switch (c) {
        case 'H':
            if (strlen(optarg) > HOST_NAME_MAX) throw std::logic_error(std::string("Hostname ") + optarg + " is longer than " + std::to_string(HOST_NAME_MAX));
            strcpy(hostname, optarg);
            break;
        case 'p': port = strtol(optarg, NULL, 10); break;
        case 'v': verbose_flag = 1; break;
        case 'V':
            print_usage(argv[0]);
            return 0;
            break;
        case 'h':
            print_usage(argv[0]);
            return 0;
            break;
        case '?': print_usage(argv[0]); return 1;
        }
    }

    if (port == 0) throw std::logic_error("Invalid port parameter");

    if (arg_index != argc - 1) {
        print_usage(argv[0]);
        return 1;
    }

    file_service = std::make_shared<LocalFileService>(hostname, argv[argc - 1]);
    file_server = std::make_shared<HTTPFileServer>(file_service, port);
    file_server->start();

    std::signal(SIGTERM, signal_handler);
    std::signal(SIGINT, signal_handler);
    std::signal(SIGQUIT, signal_handler);

    std::unique_lock<std::mutex> lock(exit_mutex);
    while (signal_status == 0) // while to avoid spurious wakeup
        exit_cond_var.wait(lock);

    file_server->stop();
    return 0;
}