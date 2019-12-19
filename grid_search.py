#!/usr/bin/env python

import os
import subprocess
import time
from threading import Thread

import distutils.spawn
psrecord_path = distutils.spawn.find_executable("psrecord")

def launch_benchmark(every, recent, filename):
    os.environ["KEEP_EVERY"] = str(every)
    os.environ["KEEP_RECENT"] = str(recent)
    output = subprocess.check_output(["make", "test-sim-benchmark"])
    with open("{}.results".format(filename), 'a') as log:
        log.write(output)

for every in range(1, 201, 50):
    for recent in range(100, 101, 100):
        filename = "every-{}-recent-{}".format(str(every), str(recent))
        print("Running {}".format(filename))

        # Some reason popen doesn't work, so we have to use check_output which waits until the command exits.
        # Therefore putting this in a separate thread.
        Thread(target=(lambda: launch_benchmark(every, recent, filename))).start()

        # Adding sleep here to wait until the benchmark command is ran from make
        time.sleep(5)

        process_pid = subprocess.check_output("ps -ef | grep simapp.test | grep -v grep | awk '{print $2}'", stderr=subprocess.STDOUT, shell=True).strip().decode("utf-8")
        print("Process pid")
        print(process_pid)
        os.system("psrecord {} --log {}.log --interval 15".format(process_pid, filename))
        os.system("{} {} --log {}.log".format(psrecord_path, str(pid), filename))
