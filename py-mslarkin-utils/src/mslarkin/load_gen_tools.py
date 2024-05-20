import math
import time
import os
import random
import sys, threading, re
from . import gcp_utils

work_break_default = 60 * 10  # Set to 15 for "infinite"
randomize_work_break_default = 'False'
cpu_intensity_default = 1
deplete_after_default = 1
max_primes_default = 25
memory_mb_default = 1
start_delay_default = 0

def get_unique_id():
    instance_id = re.search(r'.*(.{3})$', gcp_utils.get_instance_id_short()).group(1)
    process_id = str(os.getpid())
    thread_id = re.search(r'.*(.{4})$', str(threading.get_ident())).group(1)
    return instance_id + "-" + process_id + "-" + thread_id

def is_prime(number):
    for x in range(2, int(math.sqrt(number) + 1)):
        if number % x == 0:
            return False
    return True

def compute_primes(cpu_intensity, start_at=1e13, max_primes=25):
    prime_count = 0
    number = start_at if start_at % 2 == 1 else start_at - 1
    time_hack = time.time()
    while prime_count < max_primes:
        if os.environ.get("DETAILED_LOGGING", 'False') == 'True' and (time.time()-time_hack) >= 60:
            print(f"({get_unique_id()}) - Computing primes...")
            time_hack = time.time()
        if is_prime(number):
            prime_count += 1
        number += 2
        # Sleep before starting next round.  Higher->lower CPU load.
        # cpu_intensity; helps control cpu utilization
        time.sleep(cpu_intensity)
        

def consume_memory(mem_mb):
    return bytearray(round(mem_mb*1024*1024))

def get_cpu_load(cpu_intensity):
    if cpu_intensity <=.2:
        return 'high'
    elif .2 < cpu_intensity < .85:
        return 'moderate'
    else:
        return 'low'

def triggered_w_load(cpu_intensity=0, max_primes=1, memory_mb=1):
    # Set random cpu_intensity if passed 0
    if cpu_intensity == 0:
        cpu_intensity = round(random.uniform(0.01, 1), 2)
    
    print(
        f"({get_unique_id()}) - Starting triggered workload.  CPU Intensity: {cpu_intensity} ({get_cpu_load(cpu_intensity)}), max_primes: {max_primes}, memory_mb: {memory_mb}")
    start = time.time()

    # Set memory load
    memory_holder = consume_memory(memory_mb)

    # Begin CPU load
    compute_primes(cpu_intensity=cpu_intensity,
                   start_at=1e13, max_primes=max_primes)
    end = time.time()
    print(
        f"({get_unique_id()}) - Completed triggered workload.  CPU Intensity: {cpu_intensity} ({get_cpu_load(cpu_intensity)}), max_primes: {max_primes}. Workload took {end-start} seconds, using {round(sys.getsizeof(memory_holder)/1024/1024,2)}MB Memory")
    return f"Done. Completed in {end-start}s"


def triggered_no_load(sleep_time=1):
    print(f"({get_unique_id()}) - Starting triggered no-load process.  Sleep time: {sleep_time}")
    start = time.time()
    time.sleep(sleep_time)
    end = time.time()
    print(
        f"({get_unique_id()}) - Completed triggered no-load process.  Sleep time: {sleep_time}.  Completed in {end-start} seconds.")
    return f"Done. Completed in {end-start}s"


class LoadGen:
    """Class for encapsulating a persistant "do work" process."""

    def __init__(self, start_delay, work_break, cpu_intensity, deplete_after, max_primes, randomize_work_break=False, memory_mb=1):
        """Constructor.

        ARGS
          work_break (float): # seconds we want to wait before starting another
            round of work. Is the distinction between finite and infinite workloads.
          cpu_intensity (float):  # seconds we wait to do another load of work. How
            often we want to utilize cpu. Can be used to fine-tune cpu utilization
            between rounds of work.
          deplete_after (int): Number of loads of work done between rounds.
        """
        self.start_delay = start_delay
        self.work_break = work_break
        self.randomize_work_break = randomize_work_break
        self.cpu_intensity = cpu_intensity
        self.deplete_after = deplete_after
        self.max_primes = max_primes
        self.memory_mb = memory_mb

    def start_workloads(self):
        """Workload(s) starter."""
        num_workloads = 0

        while True:
            num_workloads += 1
            # Set memory load
            memory_holder = consume_memory(self.memory_mb)
            
            # do a lot of work
            start = time.time()
            compute_primes(self.cpu_intensity)
            end = time.time()
            print(f"({get_unique_id()}) - Workload took {end-start} seconds ({self.cpu_intensity}), using {round(sys.getsizeof(memory_holder)/1024/1024,2)}MB Memory")

            # check if we've done enough work loads to break out
            if num_workloads >= self.deplete_after:
                break

    def run(self):
        """Starts Infinite Workloads Loop."""
        delay_modifier = random.randint(1,3)
        modified_delay = int(self.start_delay / delay_modifier)
        print(f"({get_unique_id()}) - Delaying {modified_delay} before starting work loop.")
        time.sleep(modified_delay)
        while True:
            if self.randomize_work_break:
                delay_modifier = random.randint(1,3)
                self.work_break = int(random.randint(0, self.work_break) / delay_modifier)
            
            # Set random cpu_intensity if passed 0
            if self.cpu_intensity == 0:
                self.cpu_intensity = round(random.uniform(0.01, 1), 2)


            print(
                f"({get_unique_id()}) - Starting a work loop: CPU Intensity: {self.cpu_intensity} ({get_cpu_load(self.cpu_intensity)}), max_primes: {self.max_primes}, memory_mb: {self.memory_mb}")
            self.start_workloads()
            print(
                f"({get_unique_id()}) - Waiting for {self.work_break} seconds before starting next work loop (Random: {self.randomize_work_break})"
            )
            time.sleep(self.work_break)


def constant_loadgen(**kwargs):
    if os.environ.get("LOADGEN_RANDOMIZE_WORK_BREAK", randomize_work_break_default) == 'True':
        randomize_work_break = True
    else:
        randomize_work_break = False

    start_delay = kwargs.get('start_delay')
    if start_delay is None:
        start_delay = int(os.environ.get(
            "LOADGEN_START_DELAY", start_delay_default))
    work_break = kwargs.get('work_break')
    if work_break is None:
        work_break = int(os.environ.get(
            "LOADGEN_WORK_BREAK", work_break_default))
    cpu_intensity = kwargs.get('cpu_intensity')
    if cpu_intensity is None:
        cpu_intensity = float(os.environ.get(
            "LOADGEN_CPU_INTENSITY", cpu_intensity_default))
    deplete_after = kwargs.get('deplete_after')
    if deplete_after is None:
        deplete_after = int(os.environ.get(
            "LOADGEN_DEPLETE_AFTER", deplete_after_default))
    max_primes = kwargs.get('max_primes')
    if max_primes is None:
        max_primes = int(os.environ.get(
            "LOADGEN_MAX_PRIMES", max_primes_default))
    memory_mb = kwargs.get('memory_mb')
    if memory_mb is None:
        memory_mb = float(os.environ.get(
            "LOADGEN_MEMORY_MB", memory_mb_default))
    
    loadgen_worker = LoadGen(start_delay=start_delay, work_break=work_break, randomize_work_break=randomize_work_break,
                             cpu_intensity=cpu_intensity, deplete_after=deplete_after, max_primes=max_primes, memory_mb=memory_mb)
    return loadgen_worker
