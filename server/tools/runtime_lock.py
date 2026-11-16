"""Cross-process runtime initialization lock shared with the Go server."""
from contextlib import contextmanager
import os
import time


@contextmanager
def runtime_lock(path, timeout=10):
    # Retain the inode so waiters never lock different files for the same path.
    fd = os.open(str(path) + '.init.lock', os.O_CREAT | os.O_RDWR, 0o600)
    with os.fdopen(fd, 'r+b', buffering=0) as lock:
        deadline = time.monotonic() + timeout
        while True:
            try:
                if os.name == 'nt':
                    import msvcrt
                    lock.seek(0)
                    msvcrt.locking(lock.fileno(), msvcrt.LK_NBLCK, 1)
                else:
                    import fcntl
                    fcntl.flock(lock.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
                break
            except OSError:
                if time.monotonic() >= deadline:
                    raise TimeoutError('database initialization lock timed out')
                time.sleep(0.025)
        try:
            yield
        finally:
            if os.name == 'nt':
                lock.seek(0)
                msvcrt.locking(lock.fileno(), msvcrt.LK_UNLCK, 1)
            else:
                fcntl.flock(lock.fileno(), fcntl.LOCK_UN)
