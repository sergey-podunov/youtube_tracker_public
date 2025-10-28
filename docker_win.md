## Step 3: Set Up the Port Proxy with netsh
This is the core step. You'll use the Windows netsh utility to forward a port.
Open PowerShell or Command Prompt as an Administrator. This is critical; it will not work otherwise. (Right-click PowerShell > "Run as administrator").
Run the following command. Replace <YOUR_WINDOWS_IP> with the IP address you found in Step 2. If your port from Step 1 was not 6443, change it here as well.
``` shell
    netsh interface portproxy add v4tov4 listenport=6443 listenaddress=<YOUR_WINDOWS_IP> connectport=6443 connectaddress=127.0.0.1
```
**Example:** If your IP is `192.168.1.123`, the command is:
``` shell
    netsh interface portproxy add v4tov4 listenport=6443 listenaddress=192.168.1.123 connectport=6443 connectaddress=127.0.0.1
```

This tells Windows: "Listen for traffic coming in on `192.168.1.123:6443` and forward it to `127.0.0.1:6443`."

## Step 4: Configure Windows Firewall
Now you must create a firewall rule to allow incoming connections on that port.
1. Press Win + R, type wf.msc, and press Enter. This opens "Windows Defender Firewall with Advanced Security".
1. Click on Inbound Rules on the left.
1. Click on New Rule... on the right.
1. Select Port and click Next.
1. Select TCP and Specific local ports. Enter 6443. Click Next.
1. Select Allow the connection and click Next.
1. Choose which network profiles the rule applies to. Private is usually the correct choice for a home network. Click Next.
1. Give the rule a descriptive name, like Kubernetes API Server (Docker), and click Finish.

## Step 5: Update Your Kubeconfig on the Mac
Finally, you need to tell your Mac to connect to the Windows machine's IP address instead of localhost.
On your Mac, open your ~/.kube/config file.
Find the server line that points to the cluster.
Change localhost (or 127.0.0.1) to the IP address of your Windows machine.
Before:
``` yaml
    server: https://localhost:6443
```

**After:**
``` yaml
    server: https://192.168.1.123:6443 # <-- Use your Windows IP here
```

Save the file. Now, any kubectl command you run from your Mac will be directed to your Windows machine's Docker Desktop Kubernetes cluster.
How to Revert the Changes
If you want to undo this later, you just need to delete the port proxy and disable/delete the firewall rule.
Delete the port proxy rule (in an Administrator PowerShell):
``` shell
    netsh interface portproxy delete v4tov4 listenport=6443 listenaddress=<YOUR_WINDOWS_IP>
```

Delete the firewall rule: Open wf.msc, find the rule you created under Inbound Rules, right-click, and select "Delete".