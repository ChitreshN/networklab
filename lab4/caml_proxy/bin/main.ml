let () = print_endline "Hello, World!"

open Lwt

let counter = ref 0

let listen_address = Unix.inet_addr_loopback
let listen_port = 8080

let handle_message msg =
    match msg with
    | "read" -> (string_of_int !counter)
    | "incr" -> counter := !counter + 1;  "Incremented"
    | _ -> "Unknown command"


