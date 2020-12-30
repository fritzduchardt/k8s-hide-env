package com.fduchardt.k8shideenv.exception;

import lombok.*;

@AllArgsConstructor
@RequiredArgsConstructor
@Data
@EqualsAndHashCode(callSuper=false)
public class K8sHideEnvException extends RuntimeException {
    private final String k8sCrudId;
    private final String message;
    private String[] command;
}
