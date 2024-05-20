
import socket, pytz, os, psutil
from datetime import datetime

def open_hc_socket():
    hc_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    hc_port = int(os.environ.get('PORT', 8080))
    print(f'Binding to {hc_port}')
    with hc_socket:
        hc_socket.bind(('', hc_port))
        hc_socket.listen()
        while True:
            hc_socket.accept()

def get_pacific_timestamp(timestamp: datetime):
    pacific_tz = pytz.timezone('US/Pacific')
    return timestamp.astimezone(pacific_tz)

def get_memory_utilization():
    return round(psutil.virtual_memory().percent,2)

def get_cpu_utilization():
    return round(psutil.cpu_percent())

def get_cpu_load():
    return psutil.getloadavg()

def get_memory_total():
    total_mem = psutil.virtual_memory().total
    return round(total_mem/1e6)

def get_memory_used():
    used_mem = psutil.virtual_memory().used
    return round(used_mem/1e6)

def get_memory_available():
    available_mem = psutil.virtual_memory().available
    return round(available_mem/1e6)

def get_resource_stats():
    all_stats = {'pid':os.getpid()
                 , 'cpu_count':psutil.cpu_count()
                 , 'cpu_utilization':get_cpu_utilization()
                 , 'cpu_load':get_cpu_load()
                 , 'memory_total': get_memory_total()
                 , 'memory_available':get_memory_available()
                 , 'memory_used':get_memory_used()
                 , 'memory_utilization':get_memory_utilization()}
    return all_stats

def get_all_process_resource_stats():
    procs = psutil.process_iter()
    proc_stats = {}
    for p in procs:
        proc_stats.update({p.pid:{'name':p.name()
                                  , 'cpu_utilization': round(p.cpu_percent(),2)
                                  , 'memory_utilization': round(p.memory_percent(), 2)
                                  }})
    return proc_stats
