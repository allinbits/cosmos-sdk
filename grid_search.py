#!/usr/bin/env python

import os
import subprocess
import time
from threading import Thread

import distutils.spawn
psrecord_path = distutils.spawn.find_executable("psrecord")

EVERY_STEP = 50
RECENT_STEP = 50
START_EVERY = 1
START_RECENT = 1
END_EVERY = 501
END_RECENT = 501

class Result:
    def __init__(self, recent, sim_time, max_cpu, max_mem):
        self.recent = recent
        self.sim_time = sim_time
        self.max_cpu = max_cpu
        self.max_mem = max_mem

def launch_benchmark(every, recent, filename):
    os.environ["KEEP_EVERY"] = str(every)
    os.environ["KEEP_RECENT"] = str(recent)
    output = subprocess.check_output(["make", "test-sim-benchmark"])
    with open("{}.results".format(filename), 'a') as log:
        log.write(output)

def get_file_name(every, recent):
    return "every-{}-recent-{}".format(str(every), str(recent))

def split_line(line):
    parts = []
    for p in line.split(" "):
        if p != '':
            parts.append(p)
    return parts

def write_results_to_csv():
    results = {}

    for every in range(START_EVERY, END_EVERY, EVERY_STEP):
        results[every] = []
        for recent in range(START_RECENT, END_RECENT, RECENT_STEP):
            filename = get_file_name(every, recent)
            results_file = filename+".results"
            try:
                with open(results_file, 'r') as f:
                    lines = f.readlines()
                    parts = split_line(lines[-1])
                    sim_time = parts[1].split("\t")[-1].split("s")[0]
                    if sim_time[-1] == 's':
                        sim_time = sim_time[0:-1]
            except IOError as err:
                print("Cannot open {0}: {1}\n".format(results_file, err))
                print("Skipping to next")
                continue

            log_file = filename+".log"
            try:
                with open(log_file, 'r') as f:
                    lines = f.readlines()
                    # Skip the columns row
                    max_cpu = 0.0
                    max_mem = 0.0
                    for line in lines[1:]:
                        parts = split_line(line)
                        if float(parts[1]) > max_cpu:
                            max_cpu = float(parts[1])
                        if float(parts[2]) > max_mem:
                            max_mem = float(parts[2])
            except IOError as err:
                print("Cannot open {0}: {1}\n".format(log_file, err))
                print("Skipping to next")
                continue

            results[every].append(Result(recent, sim_time, max_cpu, max_mem))

    f = open("results.csv", "w")
    f.write("every,recent,sim_time,max_cpu,max_mem\n")
    for every, results in results.items():
        for result in results:
            f.write("{0},{1},{2},{3},{4}\n".format(every, result.recent, result.sim_time, result.max_cpu, result.max_mem))

def run_benchmarks():
    for every in range(START_EVERY, END_EVERY, EVERY_STEP):
        for recent in range(START_RECENT, END_RECENT, RECENT_STEP):
            filename = get_file_name(every, recent)
            print("Running {}".format(filename))

            # Some reason popen doesn't work, so we have to use check_output which waits until the command exits.
            # Therefore putting this in a separate thread.
            Thread(target=(lambda: launch_benchmark(every, recent, filename))).start()

            # Adding sleep here to wait until the benchmark command is ran from make
            time.sleep(5)

            process_pid = subprocess.check_output("ps -ef | grep simapp.test | grep -v grep | awk '{print $2}'", stderr=subprocess.STDOUT, shell=True).strip().decode("utf-8")
            os.system("psrecord {} --log {}.log --interval 15".format(process_pid, filename))

if __name__ == "__main__":
    run_benchmarks()
    write_results_to_csv()
