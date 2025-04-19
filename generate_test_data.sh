#!/bin/bash
# generate_test_data.sh
# Generates CSV files for JMeter testing

# Create directory for CSV files
mkdir -p jmeter_data

# Create hot assets CSV (20 assets)
echo "symbol" > jmeter_data/hot_assets.csv
head -n 21 symbols.csv | tail -n 20 >> jmeter_data/hot_assets.csv

# Create medium assets CSV (180 assets)
echo "symbol" > jmeter_data/medium_assets.csv
head -n 201 symbols.csv | tail -n 180 >> jmeter_data/medium_assets.csv

# Create cold assets CSV (200 assets)
echo "symbol" > jmeter_data/cold_assets.csv
tail -n +201 symbols.csv | head -n 200 >> jmeter_data/cold_assets.csv

# Create mixed assets CSV (100 assets from different tiers)
echo "symbol" > jmeter_data/mixed_assets.csv
head -n 11 symbols.csv | tail -n 10 >> jmeter_data/mixed_assets.csv  # 10 hot assets
head -n 101 symbols.csv | tail -n 40 >> jmeter_data/mixed_assets.csv # 40 medium assets
tail -n +201 symbols.csv | head -n 50 >> jmeter_data/mixed_assets.csv # 50 cold assets

echo "CSV files for JMeter testing created in jmeter_data directory"