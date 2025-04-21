echo "symbol" > symbols.csv
for i in {1..1000}; do
    echo "asset$i" >> symbols.csv
done