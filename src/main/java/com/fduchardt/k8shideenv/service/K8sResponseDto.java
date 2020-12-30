package com.fduchardt.k8shideenv.service;

import lombok.*;

@RequiredArgsConstructor
@AllArgsConstructor
public class K8sResponseDto {
    private final String message;
    private final String k8sCrudId;
    private String[] command;

    public String getMessage() {
        return message;
    }

    public String getK8sCrudId() {
        return k8sCrudId;
    }
}
