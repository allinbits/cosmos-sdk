#!/usr/bin/env python

import os
import subprocess

import distutils.spawn
psrecord_path = distutils.spawn.find_executable("psrecord")

for every in range(1, 501, 100):
    for recent in range(0, 201, 50):
        os.environ["KEEP_EVERY"] = str(every)
        os.environ["KEEP_RECENT"] = str(recent)
        filename = "every-{}-recent-{}".format(str(every), str(recent))
        print("Running {}".format(filename))
        log = open("{}.results".format(filename), 'a')

        env = {
            "KEEP_EVERY": str(every),
            "KEEP_RECENT": str(recent),
            "PATH": os.environ["PATH"],
            "HOME": "/home/ubuntu"
        }
        #process = subprocess.Popen(["make", "test-sim-benchmark"], shell=True, stdout=log, stderr=log, env=env)
        output = subprocess.check_output(["make", "test-sim-benchmark"])
        log.write(output)
        #pid = process.pid
        #process_pid = subprocess.check_output("ps -ef | grep simapp.test | grep -v grep | awk '{print $2}'", stderr=subprocess.STDOUT, shell=True).strip().decode("utf-8")
        #print(process_pid)
        #os.system("psrecord {} --log {}.log --interval 15".format(process_pid, filename))
        #os.system("{} {} --log {}.log".format(psrecord_path, str(pid), filename))
        log.close()
