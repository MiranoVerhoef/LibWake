<?php
// Simple settings page (works without extra includes). This is intentionally
// minimal for a first public version.

$plugin = 'libwake';
$cfgFile = "/boot/config/plugins/${plugin}/${plugin}.cfg";
$vmFile  = "/boot/config/plugins/${plugin}/vms.json";
$rcScript = "/etc/rc.d/rc.${plugin}";

function read_cfg($path) {
  $out = [];
  if (!is_file($path)) return $out;
  $lines = file($path, FILE_IGNORE_NEW_LINES);
  foreach ($lines as $line) {
    $line = trim($line);
    if ($line === '' || str_starts_with($line, '#') || str_starts_with($line, ';')) continue;
    $pos = strpos($line, '=');
    if ($pos === false) continue;
    $key = trim(substr($line, 0, $pos));
    $val = trim(substr($line, $pos+1));
    $val = trim($val, "\"'");
    $out[$key] = $val;
  }
  return $out;
}

function write_cfg($path, $kv) {
  $dir = dirname($path);
  if (!is_dir($dir)) mkdir($dir, 0775, true);
  $lines = [];
  $lines[] = '# libwake settings';
  $lines[] = 'ENABLED="'.($kv['ENABLED'] ?? 'no').'"';
  $lines[] = 'INTERFACE="'.($kv['INTERFACE'] ?? 'br0').'"';
  $lines[] = 'UDP_PORTS="'.($kv['UDP_PORTS'] ?? '7,9').'"';
  if (!empty($kv['ALLOW_SUBNETS'])) {
    $lines[] = 'ALLOW_SUBNETS="'.$kv['ALLOW_SUBNETS'].'"';
  } else {
    $lines[] = '# ALLOW_SUBNETS="192.168.1.0/24,10.0.0.0/8"';
  }
  $lines[] = 'DEBOUNCE_SECONDS="'.($kv['DEBOUNCE_SECONDS'] ?? '10').'"';
  $lines[] = 'VM_STATE_PATH="'.$kv['VM_STATE_PATH'].'"';
  file_put_contents($path, implode("\n", $lines)."\n");
}

function read_vmstate($path) {
  if (!is_file($path)) return [];
  $raw = file_get_contents($path);
  $j = json_decode($raw, true);
  if (!is_array($j)) return [];
  return $j;
}

function write_vmstate($path, $state) {
  $dir = dirname($path);
  if (!is_dir($dir)) mkdir($dir, 0775, true);
  file_put_contents($path, json_encode($state, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES)."\n");
}

function list_vms() {
  $names = trim(shell_exec('virsh list --all --name 2>/dev/null'));
  if ($names === '') return [];
  $rows = [];
  foreach (preg_split('/\R/', $names) as $name) {
    $name = trim($name);
    if ($name === '') continue;
    $uuid = trim(shell_exec('virsh domuuid '.escapeshellarg($name).' 2>/dev/null'));
    if ($uuid === '') continue;
    $rows[] = ['name'=>$name, 'uuid'=>$uuid];
  }
  usort($rows, fn($a,$b)=>strcmp($a['name'],$b['name']));
  return $rows;
}

$cfg = read_cfg($cfgFile);
if (empty($cfg['VM_STATE_PATH'])) $cfg['VM_STATE_PATH'] = $vmFile;
$vmState = read_vmstate($cfg['VM_STATE_PATH']);

$didSave = false;
if ($_SERVER['REQUEST_METHOD'] === 'POST' && ($_POST['action'] ?? '') === 'save') {
  $cfg['ENABLED'] = isset($_POST['enabled']) ? 'yes' : 'no';
  $cfg['INTERFACE'] = trim($_POST['interface'] ?? 'br0');
  $cfg['UDP_PORTS'] = trim($_POST['udp_ports'] ?? '7,9');
  $cfg['ALLOW_SUBNETS'] = trim($_POST['allow_subnets'] ?? '');
  $cfg['DEBOUNCE_SECONDS'] = trim($_POST['debounce_seconds'] ?? '10');
  $cfg['VM_STATE_PATH'] = $vmFile;

  // VM checkboxes
  $newState = [];
  foreach (list_vms() as $vm) {
    $key = 'vm_'.str_replace('-', '_', $vm['uuid']);
    $newState[$vm['uuid']] = isset($_POST[$key]);
  }

  write_cfg($cfgFile, $cfg);
  write_vmstate($vmFile, $newState);

  // Restart daemon
  if (is_file($rcScript)) {
    shell_exec(escapeshellcmd($rcScript).' restart >/dev/null 2>&1');
  }
  $vmState = $newState;
  $didSave = true;
}

$vms = list_vms();
?>

<div class="title">
  <span>LibWake</span>
</div>

<?php if ($didSave): ?>
  <div class="notice">Settings saved. Daemon restarted.</div>
<?php endif; ?>

<form method="POST">
  <input type="hidden" name="action" value="save" />

  <table class="settings">
    <tr>
      <td>Enable daemon</td>
      <td><input type="checkbox" name="enabled" <?=($cfg['ENABLED'] ?? 'no')==='yes'?'checked':''?> /></td>
    </tr>
    <tr>
      <td>Listen interface</td>
      <td><input type="text" name="interface" value="<?=htmlspecialchars($cfg['INTERFACE'] ?? 'br0')?>" /></td>
    </tr>
    <tr>
      <td>UDP ports</td>
      <td><input type="text" name="udp_ports" value="<?=htmlspecialchars($cfg['UDP_PORTS'] ?? '7,9')?>" /></td>
    </tr>
    <tr>
      <td>Allow subnets (optional)</td>
      <td><input style="width: 420px" type="text" name="allow_subnets" value="<?=htmlspecialchars($cfg['ALLOW_SUBNETS'] ?? '')?>" placeholder="192.168.1.0/24,10.0.0.0/8" /></td>
    </tr>
    <tr>
      <td>Debounce seconds</td>
      <td><input type="number" name="debounce_seconds" value="<?=htmlspecialchars($cfg['DEBOUNCE_SECONDS'] ?? '10')?>" min="0" max="300" /></td>
    </tr>
  </table>

  <h3>VMs</h3>
  <p>Select which VMs may be started by WOL packets.</p>
  <table class="settings">
    <tr><th>Enable</th><th>Name</th><th>UUID</th></tr>
    <?php foreach ($vms as $vm):
      $key = 'vm_'.str_replace('-', '_', $vm['uuid']);
      $checked = !empty($vmState[$vm['uuid']]);
    ?>
      <tr>
        <td><input type="checkbox" name="<?=$key?>" <?= $checked ? 'checked' : '' ?> /></td>
        <td><?=htmlspecialchars($vm['name'])?></td>
        <td><code><?=htmlspecialchars($vm['uuid'])?></code></td>
      </tr>
    <?php endforeach; ?>
  </table>

  <p>
    <button type="submit" class="button">Apply</button>
  </p>
</form>
