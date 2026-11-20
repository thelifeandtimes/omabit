"""Bounded serial status only; complete private reports remain on the guest disk."""
import json
import re

KIND = 'issues20-recording-compact-status-v1'
MAX_BYTES = 2048
GATES = ('original_recording_bytes_unchanged', 'policy_head_unchanged', 'actual_cli_and_stdio_measurements', 'actual_original_export_manifest_part_cli_mcp', 'actual_original_zip_matches_files', 'actual_indexed_zip_preserves_originals', 'actual_timing_query_cli_mcp', 'actual_retained_output_coverage_cli_mcp', 'actual_redacted_manifest_part_cli_mcp', 'actual_redacted_cli_zip_matches_parts')


def digest(value, size):
    return value if isinstance(value, str) and re.fullmatch('[0-9a-f]{'+str(size)+'}', value) else ''


def serialize(value):
    product = value.get('product') if isinstance(value.get('product'), dict) else {}
    source, proof = digest(value.get('source_commit'), 40), digest(value.get('proof_sha256'), 64)
    gates = {key: product.get(key) is True for key in GATES}
    helper_exit = value.get('helper_exit') if type(value.get('helper_exit')) is int and -1024 <= value['helper_exit'] <= 1024 else None
    passed = value.get('passed') is True and bool(source and proof) and helper_exit == 0 and value.get('original_rows_preserved_except_one_observation') is True and product.get('passed') is True and product.get('closed_jobs') == 2 and product.get('complete_private_proof_retained') is True and all(gates.values())
    status = {'format': 1, 'kind': KIND, 'passed': passed, 'source_commit': source, 'proof_sha256': proof, 'helper_exit': helper_exit, 'original_rows_preserved_except_one_observation': value.get('original_rows_preserved_except_one_observation') is True, 'product': gates, 'error': '' if passed else 'guest_read_checks_failed; inspect complete retained report and commands'}
    raw = json.dumps(status, separators=(',', ':'), ensure_ascii=True)
    if len(raw.encode('ascii')) > MAX_BYTES:
        raise RuntimeError('compact status exceeded fixed wire bound')
    return raw


def parse(raw, source):
    if not isinstance(raw, str) or len(raw.encode('utf-8')) > MAX_BYTES:
        raise RuntimeError('oversized/non-text compact status')
    value = json.loads(raw)
    keys = {'format', 'kind', 'passed', 'source_commit', 'proof_sha256', 'helper_exit', 'original_rows_preserved_except_one_observation', 'product', 'error'}
    if not isinstance(value, dict) or set(value) != keys or value.get('format') != 1 or value.get('kind') != KIND or type(value.get('passed')) is not bool:
        raise RuntimeError('wrong compact status protocol')
    if value.get('source_commit') and value['source_commit'] != source:
        raise RuntimeError('foreign compact status source')
    if value['passed'] and (type(value.get('helper_exit')) is not int or value['helper_exit'] != 0 or value.get('source_commit') != source or not digest(value.get('proof_sha256'), 64) or value.get('original_rows_preserved_except_one_observation') is not True or value.get('product') != {key: True for key in GATES} or value.get('error') != ''):
        raise RuntimeError('incomplete or foreign compact success status')
    return value


def read_log(text, source):
    result = None
    marker = 'OMABIT_RECORDING_USAGE_VM_RESULT '
    for line in text.splitlines():
        if marker in line:
            if result is not None:
                raise RuntimeError('duplicate compact result marker')
            result = parse(line.split(marker, 1)[1], source)
    return result
