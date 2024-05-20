Background load:  
* LOADGEN_START_DELAY - delay for initial start (default = 0)
* LOADGEN_WORK_BREAK - sets frequency of loadgen
    - DEFAULT -> 600
    - 600 -> 10min break between cycles (intermittent) 
    - 15 -> 15s break (near-constant)
* LOADGEN_CPU_INTENSITY - sets intensity of loadgen CPU usage
    - DEFAULT -> 1 (low)
    - .01 -> high CPU load
    - .35 -> moderate CPU load
    - 1.25 -> low CPU load
* LOADGEN_MAX_PRIMES - set how long the load calculation takes (important for request-based loads) 
    - DEFAULT -> 25 
    - 1 -> usually quick (seconds)
    - 25 -> ~4-5min
* LOADGEN_RANDOMIZE_WORK_BREAK - randomizes the frequency (work_break)
    - When True, work_break ranges from 15 to LOADGEN_WORK_BREAK
* LOADGEN_MEMORY_MB - memory consumption
    - DEFAULT -> 1

Request load:  
* REQUEST_CPU_INTENSITY
    - DEFAULT -> 0 (random)
    - Setting to 0 uses random range from 0.01-1.0
* REQUEST_MEMORY_MB 
    - DEFAULT -> .1
* REQUEST_MAX_PRIMES
    - DEFAULT -> 1
