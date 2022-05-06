import subprocess
import threading
import os.path

keys = ["key", "key2", "key3", "key4", "key5", "key6", "key7", "key8", "key9", "key10"]#, "key11", "key12", "key13", 
#"key14", "key15"]
vals = ["key10", "a"*20001, "key20", "key30", "b"*21, "key40", "c"*20001, "key50", "key60", "key70"]

def thread_func(ip_addr, key, value, mode, first_put):
    cmd = "bash -c "
    payload = './auto_client ' + ' ' + ip_addr + ' ' + ' ' + key + ' ' + value + ' ' + mode + ' ' + str(first_put)
    cmd = cmd + "'" + payload + "'"
    proc = subprocess.call(cmd, shell=True)


if __name__ == "__main__":
    filename = "perf_data.csv"
    if not os.path.exists(filename):
        f = open(filename, "w")
        f.write("Operation,Workload,Key,Elapsed Time,Response time,# Client,# Server,#Operations,Sleep mean time\n")
        f.close()

    # Spawn the first clients, that will setup the owners for the keys
    for i in range(0, len(keys)):
        x = threading.Thread(target=thread_func, args=('3.89.129.11', keys[i%10], vals[i%10], '2', 0))
        x.start()

    for i in range(len(keys), 30):
        key = keys[i%10]
        val =  vals[i%10]
        x = threading.Thread(target=thread_func, args=('3.89.129.11', key, val, '2', 3))
        x.start()
