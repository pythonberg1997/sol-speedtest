import json
import os

import numpy as np


def calculate_statistics(slot_list: list):
    if not slot_list:
        print("slot_list is empty, no statistics to calculate.")
        return None

    mean = np.mean(slot_list)
    median = np.median(slot_list)
    std = np.std(slot_list)
    variance = np.var(slot_list)
    min_val = np.min(slot_list)
    max_val = np.max(slot_list)

    p10 = np.percentile(slot_list, 10)
    p25 = np.percentile(slot_list, 25)
    p75 = np.percentile(slot_list, 75)
    p90 = np.percentile(slot_list, 90)
    p95 = np.percentile(slot_list, 95)
    p99 = np.percentile(slot_list, 99)

    return {
        "mean": float(mean),
        "median": float(median),
        "std": float(std),
        "variance": float(variance),
        "min": int(min_val),
        "max": int(max_val),
        "range": int(max_val - min_val),
        "p10": float(p10),
        "p25": float(p25),
        "p75": float(p75),
        "p90": float(p90),
        "p95": float(p95),
        "p99": float(p99),
    }


def process_file(file_path):
    with open(file_path, "r") as f:
        data = json.load(f)

    try:
        result = data["providerResults"][0]["urlResults"]
        if len(result) == 0:
            print(f"No result in {file_path}")
            return None

        slot_list = []
        for i in result:
            for j in i["testDetails"]:
                if j.get("slotDelta"):
                    slot_list.append(j["slotDelta"])

        if not slot_list:
            print(f"No slot deltas found in {file_path}")
            return None

        stats = calculate_statistics(slot_list)
        return stats
    except (KeyError, IndexError) as e:
        print(f"Error processing {file_path}: {e}")
        return None


def main():
    results_dir = "results"

    if not os.path.exists(results_dir):
        print(f"Results directory '{results_dir}' not found!")
        return

    file_list = os.listdir(results_dir)
    file_list = [file for file in file_list if file.endswith(".json")]
    file_list.sort()

    if not file_list:
        print(f"No JSON files found in '{results_dir}'")
        return

    print(f"Found {len(file_list)} JSON files to process")

    all_stats = {}
    for file_name in file_list:
        file_path = os.path.join(results_dir, file_name)
        print(f"\nProcessing {file_name}...")

        prefix = file_name[:-5]

        stats = process_file(file_path)
        if not stats:
            print(f"No statistics generated for {file_name}")
            continue

        all_stats[prefix] = stats
        print(f"Successfully processed {file_name}")

    output_file = "slot_statistics.json"
    with open(output_file, "w") as f:
        json.dump(all_stats, f, indent=2)

    print(f"\nStatistics saved to {output_file}")
    print(f"Prefixes found: {list(all_stats.keys())}")


if __name__ == "__main__":
    main()
