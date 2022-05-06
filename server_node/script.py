import subprocess
import threading
import time

def thread_func_server(ip_addr, nil):
    cmd = "bash -c "
    payload = './server_node ' + ' ' + ip_addr
    cmd = cmd + "'" + payload + "'"
    proc = subprocess.call(cmd, shell=True)


if __name__ == "__main__":
    for i in range(0,15):
        x = threading.Thread(target=thread_func_server, args=('172.28.165.24', 0))
        x.start()
        time.sleep(0.2)
