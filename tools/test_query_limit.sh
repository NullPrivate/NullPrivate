# run query 120 times quickly
for i in {1..120}; do
    echo "Query $i"
    # ./tools/q -s 127.0.0.1:5353 -t A -S -q baidu.com
    # ./tools/q -s 127.0.0.1:5353 -t A -w -q baidu.com
    # ./tools/q -s 127.0.0.1:5353 -t A --timeout 10s -q example.com
    # reverse query
    # ./tools/q -s 127.0.0.1:5353 -x -t PTR -q 127.0.0.1
    # dot query
    # ./tools/q -s tls://dns.jqknono.com:34026 -t A --timeout 10s -q example.com
    # doq query
    # ./tools/q -s quic://dns.jqknono.com:34027 -t A --timeout 10s -q example.com
    # doh query
    ./tools/q -s https://dns.jqknono.com:34025/dns-query -t A --timeout 10s -q example.com
done
