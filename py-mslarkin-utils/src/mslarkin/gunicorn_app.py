import os
import multiprocessing
import os, signal
from gunicorn.app.wsgiapp import WSGIApplication

class GunicornApp(WSGIApplication):
    def __init__(self, app_uri, options=None):
        self.options = options or {}
        self.app_uri = app_uri
        super().__init__()

    def load_config(self):
        config = {
            key: value
            for key, value in self.options.items()
            if key in self.cfg.settings and value is not None
        }
        for key, value in config.items():
            self.cfg.set(key.lower(), value)


def hello_world():
    return "Hello!"

def create_app(app_uri='main:app', **kwargs):
    env_port = os.environ.get("PORT", 8080)
    options = kwargs
    options.setdefault("port", env_port)
    print(f'Creating gunicorn app {app_uri} with {options}')
    return GunicornApp(app_uri, options)

def get_workers():
    return multiprocessing.cpu_count() * 2 + 1

def kill_instance():
    print(f'Manually killing instance')
    os.kill(1, signal.SIGTERM)
    return "Done" 

def health_check():
    # print('Health check called')
    return 'Done'