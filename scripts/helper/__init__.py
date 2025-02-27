''' Helper functions that are use by, both, test.py and wasm-test.py '''

#pylint: disable=too-few-public-methods, c-extension-no-member, broad-except, global-statement, missing-function-docstring, missing-class-docstring, consider-using-with, fixme

from subprocess import check_output, Popen
from time import sleep
from collections import OrderedDict
from sys import stdout
from os.path import exists
from os import remove

import copy
import subprocess
import os
import json
import requests

_OL_DIR = None
_CURR_CONF = None

def setup_config(ol_dir):
    global _OL_DIR
    _OL_DIR = ol_dir

def get_ol_stats():
    if os.path.exists(f"{_OL_DIR}/worker/stats.json"):
        with open(f"{_OL_DIR}/worker/stats.json", "r", encoding='utf-8') as statsfile:
            olstats = json.load(statsfile)
        return OrderedDict(sorted(list(olstats.items())))

    return None

def get_worker_output():
    if _OL_DIR is not None:
        with open(os.path.join(_OL_DIR, "worker.out"), "r", encoding='utf-8') as workerfile:
            return workerfile.read().splitlines()
    else:
        # The WebAssembly worker does not create an output currently
        # TODO we should capture it somehow
        return ["_OL_DIR not set"]

def get_current_config():
    return _CURR_CONF

def post(path, data=None):
    ''' Issues a post request to the OL worker '''
    return requests.post(f'http://localhost:5000/{path}', json.dumps(data))

def put_conf(conf):
    global _CURR_CONF
    with open(os.path.join(_OL_DIR, "config.json"), 'w', encoding='utf-8') as cfile:
        json.dump(conf, cfile, indent=2)
    _CURR_CONF = conf

class TestConf:
    ''' Loads a config and overwrites certain fields with what is set in **keywords '''
    def __init__(self, **keywords):
        self.orig = None

        with open(os.path.join(_OL_DIR, "config.json"), "r", encoding='utf-8') as cfile:
            try:
                self.orig = json.load(cfile)
            except json.JSONDecodeError as err:
                raise Exception(
                    f"Failed to parse JSON file. Contents are:\n"
                    f"{cfile.read()}"
                ) from err

        new = copy.deepcopy(self.orig)
        for (key, value) in keywords.items():
            if not key in new:
                raise Exception(f"unknown config param: {key}")

            if isinstance(value, dict):
                for key2 in value:
                    new[key][key2] = value[key2]
            else:
                new[key] = value

        # setup
        put_conf(new)

    def cleanup(self):
        put_conf(self.orig)

class TestConfContext:
    def __init__(self, **keywords):
        self._conf = None
        self._keywords = keywords

    def __enter__(self):
        self._conf = TestConf(**self._keywords)

    def __exit__(self, _exc_type, _exc_value, _exc_traceback):
        self._conf.cleanup()

def run(cmd):
    print("RUN", " ".join(cmd))
    try:
        out = check_output(cmd, stderr=subprocess.STDOUT)
        fail = False
    except subprocess.CalledProcessError as err:
        out = err.output
        fail = True

    out = str(out, 'utf-8')
    if len(out) > 500:
        out = out[:500] + "..."

    if fail:
        raise Exception(f"command ({' '.join(cmd)}) failed: {out}")
    print(out)

class DockerWorker():
    def __init__(self):
        self._running = False
        self._config = TestConf(sandbox="docker", features={"import_cache": False})

        try:
            print("Starting Docker container worker")
            run(['./ol', 'worker', f'-p={_OL_DIR}', '--detach'])
        except Exception as err:
            raise RuntimeError(f"failed to start worker: {err}") from err

        self._running = True

    def __del__(self):
        self.stop()

    def is_running(self):
        return self._running

    @staticmethod
    def name():
        return "docker"

    def stop(self):
        if self.is_running():
            self._running = False
        else:
            return # Already stopped

        try:
            print("Stopping Docker container worker")
            run(['./ol', 'kill', '-p='+_OL_DIR])
        except Exception as err:
            raise RuntimeError("Failed to start worker") from err

class SockWorker():
    def __init__(self):
        self._running = False
        self._config = TestConf(sandbox="sock")

        try:
            print("Starting SOCK container worker")
            run(['./ol', 'worker', '-p='+_OL_DIR, '--detach'])
        except Exception as err:
            raise RuntimeError(f"failed to start worker: {err}") from err

        self._running = True

    def __del__(self):
        self.stop()

    def is_running(self):
        return self._running

    @staticmethod
    def name():
        return "SOCK"

    def stop(self):
        if self.is_running():
            self._running = False
        else:
            return # Already stopped

        try:
            print("Stopping SOCK container worker")
            run(['./ol', 'kill', '-p='+_OL_DIR])
        except Exception as err:
            raise RuntimeError("Failed to start worker") from err

class WasmWorker():
    def __init__(self):
        # TODO use a similar mechanism to regular OpenLambda

        stdout.write("Starting WebAssembly worker.")
        stdout.flush()

        if exists('./ol-wasm.ready'):
            remove('./ol-wasm.ready')

        self._process = Popen(["./ol-wasm"])

        while not exists('./ol-wasm.ready'):
            sleep(0.5)
            stdout.write('.')
            stdout.flush()

        print("Done")

    def __del__(self):
        self.stop()

    def is_running(self):
        return self._process is not None

    @staticmethod
    def name():
        return "wasm"

    @staticmethod
    def run(fn_name, args=None):
        result = post(f"run/{fn_name}", data=args)

        if result.status_code != 200:
            raise RuntimeError(f"Benchmark was not successful: {result.text}")

    def stop(self):
        if not self.is_running():
            return

        print("Stopping WebAssembly worker")
        self._process.terminate()
        self._process = None

def prepare_open_lambda(ol_dir, reuse_config=False):
    '''
    Sets up the working director for open lambda,
    and stops currently running worker processes (if any)
    '''
    if os.path.exists(_OL_DIR):
        try:
            run(['./ol', 'kill', f'-p={ol_dir}'])
            print("stopped existing worker")
        except Exception as err:
            print(f"Could not kill existing worker: {err}")

    # general setup
    if not reuse_config:
        if os.path.exists(ol_dir):
            run(['rm', '-rf', ol_dir])

        run(['./ol', 'new', f'-p={ol_dir}'])
    else:
        if os.path.exists(_OL_DIR):
            # Make sure the pid file is gone even if the previous worker crashed
            try:
                run(['rm', '-rf', f'{ol_dir}/worker'])
            except Exception as _:
                pass
        else:
            # There was never a config in the first place, create one
            run(['./ol', 'new', f'-p={ol_dir}'])

def mounts():
    ''' Returns a list of all mounted directories '''

    output = check_output(["mount"])
    output = str(output, "utf-8")
    output = output.split("\n")
    return set(output)

def ol_oom_killer():
    ''' Will terminate OpenLambda if we run out of memory '''

    while True:
        if get_mem_stat_mb('MemAvailable') < 128:
            print("out of memory, trying to kill OL")
            os.system('pkill ol')
        sleep(1)

def get_mem_stat_mb(stat):
    with open('/proc/meminfo', 'r', encoding='utf-8') as memfile:
        for line in memfile:
            if line.startswith(stat+":"):
                parts = line.strip().split()
                assert_eq(parts[-1], 'kB')
                return int(parts[1]) / 1024
    raise Exception('could not get stat')

def assert_eq(actual, expected):
    ''' Test helper. Will fail if actual != expected '''

    if expected != actual:
        raise Exception(f'Expected value "{expected}", but was "{actual}"')
