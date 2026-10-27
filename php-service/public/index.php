<?php

declare(strict_types=1);

// Front controller: SEMUA request diarahkan nginx ke file ini.
//
// Beda paling mendasar dengan Go/Rust: PHP-FPM bersifat "shared-nothing".
// Setiap request memulai skrip ini dari nol — objek, variabel, dan "state"
// dibuang saat request selesai. Tidak ada main() yang hidup terus.
// Opcache menyimpan bytecode yang sudah dikompilasi supaya tidak di-parse ulang,
// dan APCu (shared memory) dipakai untuk data yang harus bertahan antar request.

require __DIR__ . '/../vendor/autoload.php';

App\App::handle();
