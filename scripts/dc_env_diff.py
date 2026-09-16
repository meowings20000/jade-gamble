
import hashlib, subprocess, os, re
env = dict(l.strip().split("=", 1) for l in open(r"C:\jade-gamble\.env", encoding="utf-8") if "=" in l and not l.strip().startswith("#"))
out = subprocess.run(["docker", "exec", "jade-gamble", "env"], capture_output=True, text=True).stdout
cont = dict(l.split("=", 1) for l in out.splitlines() if "=" in l)
def h(v): return hashlib.sha256(v.encode()).hexdigest()[:12] if v else "(empty)"
for k in ["DISCORD_CLIENT_ID", "DISCORD_CLIENT_SECRET", "DISCORD_REDIRECT_URI", "DISCORD_GUILD_ID"]:
    a = env.get(k, ""); b = cont.get(k, "")
    print(f"{k}:")
    print(f"   .env      len={len(a)} sha={h(a)}  head={a[:12]}")
    print(f"   container len={len(b)} sha={h(b)}  head={b[:12]}")
    print(f"   {'MATCH' if a.strip()==b.strip() else '!!! MISMATCH !!!'}")
print("\ndocker log tail:")
print(subprocess.run(["docker", "logs", "--tail", "12", "jade-gamble"], capture_output=True, text=True).stderr[-1200:])
