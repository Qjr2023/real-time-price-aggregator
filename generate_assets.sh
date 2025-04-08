echo "symbol" > symbols.csv
for i in {1..10000}; do
    echo "asset$i" >> symbols.csv
done