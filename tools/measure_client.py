#!/usr/bin/env python3
"""Run isolated, interleaved GX client GL measurements with local assets.

Source tools/go-env.sh first. Results/captures/profiles are private artifacts;
no server, gameplay settings or database is changed. This measures hidden-window
render throughput, not presented FPS or end-to-end server/network latency.
"""
import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import random
import statistics
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_CASES = 'world-stream,world-mobs,world-players,world-effect,world-hud,water-visible,dungeon,crowd-200,crowd-400,crowd-effects,crowd-network'


def command(args, **kwargs):
    return subprocess.check_output(args, text=True, **kwargs).strip()


def gpu_snapshot():
    try:
        rows = command(['nvidia-smi', '--query-gpu=name,driver_version,temperature.gpu,pstate,clocks.current.graphics,utilization.gpu,memory.used', '--format=csv,noheader,nounits'], timeout=10)
        processes = command(['nvidia-smi', 'pmon', '-c', '1'], timeout=10)
        return {'available': True, 'devices': rows, 'processes': processes}
    except (OSError, subprocess.SubprocessError) as err:
        return {'available': False, 'reason': str(err)}


def summarize(values):
    values = sorted(values)
    if not values:
        return {}
    import math
    return {'samples': len(values), 'mean_ms': statistics.fmean(values), **{
        f'p{percent}_ms': values[max(0, math.ceil(len(values)*percent/100)-1)]
        for percent in (50, 95, 99)}, 'max_ms': values[-1]}


def report(output, runs):
    grouped = {}
    for run in runs:
        grouped.setdefault((run['scene'], run['variant']), []).append(run)
    lines = ['# GX client CPU/render measurements', '',
             f"UTC: {datetime.now(timezone.utc).isoformat()}", '',
             'Hidden GL 4.5 window, VSync off, no presentation, fixed simulation step 1/60 s, at most three GPU frames in flight. CPU submission is wall time in packet pumping + Update + Draw; it includes driver work/waits inside those calls. GPU elapsed timestamps are distinct per frame and may include submission gaps. Explicit queue waits are reported separately. These numbers are not presented FPS.', '',
             'Every run uses isolated preferences and local read-only assets. Setup/loading frames, steady-state frames, screenshots and profiles are separate. Synthetic packet input begins after socket decoding; it does not measure TCP, TLS or server latency.', '']
    if runs:
        lines += [f"Renderer: {runs[0]['renderer']}; {runs[0]['gl_version']}; {runs[0]['go_version']}.", '',
                  f"GPU contention classification: **{runs[0]['load_label']}**. Other programs were not stopped by this tool.", '']
    lines += ['| Scene / variant | Runs | Actors loaded | CPU p50 / p95 / p99 ms | GPU mean ms | Explicit wait p95 ms | Draws | CPU p50 range between runs |', '|---|---:|---:|---|---:|---:|---:|---|']
    summary = []
    for (scene, variant), cases in sorted(grouped.items()):
        cpu = summarize([u+d+p for r in cases for u,d,p in zip(r['samples_ms']['update'], r['samples_ms']['draw'], r['samples_ms']['pump'])])
        gpu = summarize([v for r in cases for v in r['samples_ms']['gpu'] if v >= 0])
        wait = summarize([v for r in cases for v in r['samples_ms']['gpu_wait']])
        medians = [r['cpu_submission']['p50_ms'] for r in cases]
        lines += [f"| {scene} / {variant} | {len(cases)} | {cases[0]['loaded_remote_actors']} | {cpu['p50_ms']:.3f} / {cpu['p95_ms']:.3f} / {cpu['p99_ms']:.3f} | {gpu.get('mean_ms', 0):.3f} | {wait['p95_ms']:.3f} | {statistics.fmean(r['draw_calls_mean'] for r in cases):.0f} | {min(medians):.3f}–{max(medians):.3f} ms |"]
        summary.append({'scene': scene, 'variant': variant, 'runs': len(cases), 'cpu_submission': cpu, 'gpu_elapsed': gpu, 'queue_wait': wait, 'run_cpu_p50_range_ms': [min(medians), max(medians)]})
    lines += ['', '## Update and synthetic packet costs', '', '| Scene / variant | Update mean / p99 ms | Pump mean / p99 ms | Payloads/run | Allocs/frame |', '|---|---|---|---:|---:|']
    for (scene, variant), cases in sorted(grouped.items()):
        update = summarize([v for r in cases for v in r['samples_ms']['update']])
        pump = summarize([v for r in cases for v in r['samples_ms']['pump']])
        lines += [f"| {scene} / {variant} | {update['mean_ms']:.3f} / {update['p99_ms']:.3f} | {pump['mean_ms']:.3f} / {pump['p99_ms']:.3f} | {cases[0]['packets_applied']} | {statistics.fmean(r['allocs_per_frame'] for r in cases):.0f} |"]
    lines += ['', '## Interpretation limits', '',
              '- Loading samples are application-cold but the OS filesystem/driver caches may already be warm; this is not a cold-boot disk benchmark.',
              '- GPU clocks, temperatures and other processes are recorded before/after each run; contention invalidates standalone baseline comparisons.',
              '- Stress actors have fixed placements and a mix of visible/distant users, monsters and NPCs. They do not reproduce a particular live city.',
              '- Animation/effect time advances. Captures are diagnostic; pixel equality is not asserted for random/time-dependent effects.',
              '- No-glow/shadow/water variants are diagnostic ablations, not optimization implementations or additive savings.',
              '- Profiles are separate from reported unprofiled runs; do not combine their timing distributions.', '']
    (output/'report.md').write_text('\n'.join(lines))
    (output/'summary.json').write_text(json.dumps(summary, indent=2))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--data', required=True, type=Path)
    parser.add_argument('--output', type=Path)
    parser.add_argument('--cases', default=DEFAULT_CASES, help='scene or scene:variant, comma-separated')
    parser.add_argument('--frames', type=int, default=900)
    parser.add_argument('--repetitions', type=int, default=3)
    parser.add_argument('--width', type=int, default=1280)
    parser.add_argument('--height', type=int, default=720)
    parser.add_argument('--load-label', choices=('unverified', 'shared', 'quiet'), default='unverified', help='Operator classification, retained with telemetry; never inferred as exclusive hardware ownership')
    parser.add_argument('--binary', type=Path, help='Reuse an explicitly built current test binary')
    parser.add_argument('--profile', default='', help='Additional scene:variant cases to profile separately')
    args = parser.parse_args()
    if not args.data.is_dir():
        parser.error('--data must be the local loose data directory')
    if not 30 <= args.frames <= 30000 or not 1 <= args.repetitions <= 10:
        parser.error('frames must be 30..30000 and repetitions 1..10')
    output = (args.output or ROOT/'work/client-measure'/datetime.now().strftime('%Y%m%d-%H%M%S')).resolve()
    if output.exists() and any(output.iterdir()):
        parser.error('output must be a new or empty directory; existing artifacts are never overwritten')
    output.mkdir(mode=0o700, parents=True, exist_ok=True)
    binary = args.binary.resolve() if args.binary else output/'app.test'
    if not args.binary:
        print('Building Linux integration measurement binary…', flush=True)
        subprocess.run(['go', 'test', '-tags', 'integration', '-c', '-o', str(binary), './internal/app'], cwd=ROOT, check=True)
    digest = hashlib.sha256()
    # The production source identity is independent of benchmark-only commits.
    for base in ('internal', 'cmd/shaiya'):
        for path in sorted((ROOT/base).rglob('*.go')):
            if path.name.endswith('_test.go'):
                continue
            digest.update(str(path.relative_to(ROOT)).encode()); digest.update(path.read_bytes())
    manifest = {'schema': 1, 'utc': datetime.now(timezone.utc).isoformat(), 'commit': command(['git', 'rev-parse', 'HEAD'], cwd=ROOT),
                'production_source_scope': ['internal/**/*.go excluding tests', 'cmd/shaiya/**/*.go excluding tests'], 'production_source_sha256': digest.hexdigest(), 'measurement_fixture_sha256': hashlib.sha256((ROOT/'internal/app/performance_baseline_integration_test.go').read_bytes()).hexdigest(), 'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(),
                'arguments': {k: str(v) if isinstance(v, Path) else v for k,v in vars(args).items()},
                'cpu': command(['uname', '-m']), 'cpu_model': next((line.split(':',1)[1].strip() for line in Path('/proc/cpuinfo').read_text().splitlines() if line.startswith('model name')), 'unknown'), 'gpu_initial': gpu_snapshot(), 'runs': []}
    (output/'manifest.json').write_text(json.dumps(manifest, indent=2))
    cases = [tuple((case+':default').split(':')[:2]) for case in args.cases.split(',') if case]
    runs = []
    def run_case(scene, variant, label, profile=False):
        destination = output/label
        destination.mkdir(mode=0o700)
        before = gpu_snapshot()
        env = dict(os.environ, GX_MEASURE_CLIENT='1', SHAIYA_TEST_RENDER='1', SHAIYA_DATA=str(args.data.resolve()), GX_MEASURE_SCENE=scene,
                   GX_MEASURE_VARIANT=variant, GX_MEASURE_FRAMES=str(max(args.frames, 3000) if profile else args.frames),
                   GX_MEASURE_WIDTH=str(args.width), GX_MEASURE_HEIGHT=str(args.height), GX_MEASURE_OUTPUT=str(destination))
        env.pop('GX_MEASURE_CPU_PROFILE', None)
        if profile:
            env['GX_MEASURE_CPU_PROFILE'] = str(destination/'cpu.pprof')
        print(f'{label}: {scene}/{variant}', flush=True)
        started = time.monotonic()
        with (destination/'run.log').open('w') as log:
            completed = subprocess.run([str(binary), '-test.run', '^TestClientPerformanceBaseline$', '-test.v', '-test.timeout=300s'], cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=330)
        if completed.returncode:
            raise RuntimeError(f'{label} failed; inspect {destination / "run.log"}')
        data = json.loads((destination/'measurement.json').read_text())
        data.update(load_label=args.load_label, gpu_before=before, gpu_after=gpu_snapshot(), process_wall_seconds=time.monotonic()-started)
        (destination/'measurement.json').write_text(json.dumps(data, indent=2))
        manifest['runs'].append({'path': label, 'scene': scene, 'variant': variant, 'profiled': profile})
        (output/'manifest.json').write_text(json.dumps(manifest, indent=2))
        if profile:
            for mode in ('-top', '-cum'):
                with (destination/f'pprof{mode}.txt').open('w') as log:
                    subprocess.run(['go', 'tool', 'pprof', '-top'] + (['-cum'] if mode == '-cum' else []) + [str(binary), str(destination/'cpu.pprof')], stdout=log, stderr=subprocess.STDOUT, check=True)
        else:
            runs.append(data); report(output, runs)
        print(f"  CPU p50/p95/p99 {data['cpu_submission']['p50_ms']:.3f}/{data['cpu_submission']['p95_ms']:.3f}/{data['cpu_submission']['p99_ms']:.3f} ms; GPU {data['gpu_elapsed']['samples']}/{data['frames']} samples", flush=True)
    try:
        for repetition in range(args.repetitions):
            order = list(cases)
            random.Random(1701+repetition).shuffle(order)
            for scene, variant in order:
                run_case(scene, variant, f'r{repetition+1}-{scene}-{variant}')
        for case in filter(None, args.profile.split(',')):
            scene, variant = (case+':default').split(':')[:2]
            run_case(scene, variant, f'profile-{scene}-{variant}', profile=True)
    finally:
        report(output, runs)
    print(f'Report: {output / "report.md"}', flush=True)


if __name__ == '__main__':
    main()
