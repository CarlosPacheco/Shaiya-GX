import multiprocessing
from pathlib import Path
import tempfile
import unittest
from runtime_lock import runtime_lock


def hold(path, ready):
    with runtime_lock(path):
        ready.set()
        import time
        time.sleep(60)


class RuntimeLockTests(unittest.TestCase):
    def test_exclusion_and_process_exit_release(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'runtime.sqlite'
            ctx = multiprocessing.get_context('spawn')
            ready = ctx.Event()
            worker = ctx.Process(target=hold, args=(path, ready))
            worker.start()
            try:
                self.assertTrue(ready.wait(10))
                with self.assertRaises(TimeoutError):
                    with runtime_lock(path, timeout=0.05):
                        self.fail('acquired another process lock')
            finally:
                worker.terminate()
                worker.join(10)
            with runtime_lock(path, timeout=1):
                self.assertTrue(Path(str(path) + '.init.lock').is_file())
